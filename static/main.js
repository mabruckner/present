app = angular.module("App", function(){});

app.controller("View",['$scope','$http','$interval',function($scope,$http,$interval) {
    $scope.thing = 'hello';
    $scope.slideindex = 0;
    $scope.iframe = "/slide?index=0";
    $interval(function(){
        $http.get('/control').then(function(data){
            $scope.iframe="/slide?index="+data.data;
        },angular.noop);
    },500);

}]);
