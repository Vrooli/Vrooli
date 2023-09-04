export const auth_emailLogIn = {
  "fieldName": "emailLogIn",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "emailLogIn",
        "loc": {
          "start": 327,
          "end": 337
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 338,
              "end": 343
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 346,
                "end": 351
              }
            },
            "loc": {
              "start": 345,
              "end": 351
            }
          },
          "loc": {
            "start": 338,
            "end": 351
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLoggedIn",
              "loc": {
                "start": 359,
                "end": 369
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 359,
              "end": 369
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 374,
                "end": 382
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 374,
              "end": 382
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 387,
                "end": 392
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "activeFocusMode",
                    "loc": {
                      "start": 403,
                      "end": 418
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "mode",
                          "loc": {
                            "start": 433,
                            "end": 437
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filters",
                                "loc": {
                                  "start": 456,
                                  "end": 463
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 486,
                                        "end": 488
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 486,
                                      "end": 488
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "filterType",
                                      "loc": {
                                        "start": 509,
                                        "end": 519
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 509,
                                      "end": 519
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 540,
                                        "end": 543
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 570,
                                              "end": 572
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 570,
                                            "end": 572
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 597,
                                              "end": 607
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 597,
                                            "end": 607
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tag",
                                            "loc": {
                                              "start": 632,
                                              "end": 635
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 632,
                                            "end": 635
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 660,
                                              "end": 669
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 660,
                                            "end": 669
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 694,
                                              "end": 706
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 737,
                                                    "end": 739
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 737,
                                                  "end": 739
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 768,
                                                    "end": 776
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 768,
                                                  "end": 776
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 805,
                                                    "end": 816
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 805,
                                                  "end": 816
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 707,
                                              "end": 842
                                            }
                                          },
                                          "loc": {
                                            "start": 694,
                                            "end": 842
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 867,
                                              "end": 870
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isOwn",
                                                  "loc": {
                                                    "start": 901,
                                                    "end": 906
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 901,
                                                  "end": 906
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 935,
                                                    "end": 947
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 935,
                                                  "end": 947
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 871,
                                              "end": 973
                                            }
                                          },
                                          "loc": {
                                            "start": 867,
                                            "end": 973
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 544,
                                        "end": 995
                                      }
                                    },
                                    "loc": {
                                      "start": 540,
                                      "end": 995
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "focusMode",
                                      "loc": {
                                        "start": 1016,
                                        "end": 1025
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "labels",
                                            "loc": {
                                              "start": 1052,
                                              "end": 1058
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 1089,
                                                    "end": 1091
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1089,
                                                  "end": 1091
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "color",
                                                  "loc": {
                                                    "start": 1120,
                                                    "end": 1125
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1120,
                                                  "end": 1125
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "label",
                                                  "loc": {
                                                    "start": 1154,
                                                    "end": 1159
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1154,
                                                  "end": 1159
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1059,
                                              "end": 1185
                                            }
                                          },
                                          "loc": {
                                            "start": 1052,
                                            "end": 1185
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderList",
                                            "loc": {
                                              "start": 1210,
                                              "end": 1222
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 1253,
                                                    "end": 1255
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1253,
                                                  "end": 1255
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 1284,
                                                    "end": 1294
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1284,
                                                  "end": 1294
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 1323,
                                                    "end": 1333
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1323,
                                                  "end": 1333
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminders",
                                                  "loc": {
                                                    "start": 1362,
                                                    "end": 1371
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 1406,
                                                          "end": 1408
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1406,
                                                        "end": 1408
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 1441,
                                                          "end": 1451
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1441,
                                                        "end": 1451
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 1484,
                                                          "end": 1494
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1484,
                                                        "end": 1494
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 1527,
                                                          "end": 1531
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1527,
                                                        "end": 1531
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 1564,
                                                          "end": 1575
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1564,
                                                        "end": 1575
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 1608,
                                                          "end": 1615
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1608,
                                                        "end": 1615
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 1648,
                                                          "end": 1653
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1648,
                                                        "end": 1653
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 1686,
                                                          "end": 1696
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1686,
                                                        "end": 1696
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "reminderItems",
                                                        "loc": {
                                                          "start": 1729,
                                                          "end": 1742
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "selectionSet": {
                                                        "kind": "SelectionSet",
                                                        "selections": [
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "id",
                                                              "loc": {
                                                                "start": 1781,
                                                                "end": 1783
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1781,
                                                              "end": 1783
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "created_at",
                                                              "loc": {
                                                                "start": 1820,
                                                                "end": 1830
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1820,
                                                              "end": 1830
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "updated_at",
                                                              "loc": {
                                                                "start": 1867,
                                                                "end": 1877
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1867,
                                                              "end": 1877
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 1914,
                                                                "end": 1918
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1914,
                                                              "end": 1918
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 1955,
                                                                "end": 1966
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1955,
                                                              "end": 1966
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "dueDate",
                                                              "loc": {
                                                                "start": 2003,
                                                                "end": 2010
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2003,
                                                              "end": 2010
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "index",
                                                              "loc": {
                                                                "start": 2047,
                                                                "end": 2052
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2047,
                                                              "end": 2052
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "isComplete",
                                                              "loc": {
                                                                "start": 2089,
                                                                "end": 2099
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2089,
                                                              "end": 2099
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 1743,
                                                          "end": 2133
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 1729,
                                                        "end": 2133
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 1372,
                                                    "end": 2163
                                                  }
                                                },
                                                "loc": {
                                                  "start": 1362,
                                                  "end": 2163
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1223,
                                              "end": 2189
                                            }
                                          },
                                          "loc": {
                                            "start": 1210,
                                            "end": 2189
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resourceList",
                                            "loc": {
                                              "start": 2214,
                                              "end": 2226
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 2257,
                                                    "end": 2259
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2257,
                                                  "end": 2259
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 2288,
                                                    "end": 2298
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2288,
                                                  "end": 2298
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 2327,
                                                    "end": 2339
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 2374,
                                                          "end": 2376
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2374,
                                                        "end": 2376
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 2409,
                                                          "end": 2417
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2409,
                                                        "end": 2417
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 2450,
                                                          "end": 2461
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2450,
                                                        "end": 2461
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 2494,
                                                          "end": 2498
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2494,
                                                        "end": 2498
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2340,
                                                    "end": 2528
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2327,
                                                  "end": 2528
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "resources",
                                                  "loc": {
                                                    "start": 2557,
                                                    "end": 2566
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 2601,
                                                          "end": 2603
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2601,
                                                        "end": 2603
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 2636,
                                                          "end": 2641
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2636,
                                                        "end": 2641
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "link",
                                                        "loc": {
                                                          "start": 2674,
                                                          "end": 2678
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2674,
                                                        "end": 2678
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "usedFor",
                                                        "loc": {
                                                          "start": 2711,
                                                          "end": 2718
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2711,
                                                        "end": 2718
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "translations",
                                                        "loc": {
                                                          "start": 2751,
                                                          "end": 2763
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "selectionSet": {
                                                        "kind": "SelectionSet",
                                                        "selections": [
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "id",
                                                              "loc": {
                                                                "start": 2802,
                                                                "end": 2804
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2802,
                                                              "end": 2804
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "language",
                                                              "loc": {
                                                                "start": 2841,
                                                                "end": 2849
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2841,
                                                              "end": 2849
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 2886,
                                                                "end": 2897
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2886,
                                                              "end": 2897
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 2934,
                                                                "end": 2938
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2934,
                                                              "end": 2938
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 2764,
                                                          "end": 2972
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 2751,
                                                        "end": 2972
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2567,
                                                    "end": 3002
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2557,
                                                  "end": 3002
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2227,
                                              "end": 3028
                                            }
                                          },
                                          "loc": {
                                            "start": 2214,
                                            "end": 3028
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 3053,
                                              "end": 3061
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "FragmentSpread",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "Schedule_common",
                                                  "loc": {
                                                    "start": 3095,
                                                    "end": 3110
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 3092,
                                                  "end": 3110
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3062,
                                              "end": 3136
                                            }
                                          },
                                          "loc": {
                                            "start": 3053,
                                            "end": 3136
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3161,
                                              "end": 3163
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3161,
                                            "end": 3163
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3188,
                                              "end": 3192
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3188,
                                            "end": 3192
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3217,
                                              "end": 3228
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3217,
                                            "end": 3228
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1026,
                                        "end": 3250
                                      }
                                    },
                                    "loc": {
                                      "start": 1016,
                                      "end": 3250
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 464,
                                  "end": 3268
                                }
                              },
                              "loc": {
                                "start": 456,
                                "end": 3268
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 3285,
                                  "end": 3291
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3314,
                                        "end": 3316
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3314,
                                      "end": 3316
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 3337,
                                        "end": 3342
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3337,
                                      "end": 3342
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 3363,
                                        "end": 3368
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3363,
                                      "end": 3368
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3292,
                                  "end": 3386
                                }
                              },
                              "loc": {
                                "start": 3285,
                                "end": 3386
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 3403,
                                  "end": 3415
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3438,
                                        "end": 3440
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3438,
                                      "end": 3440
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3461,
                                        "end": 3471
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3461,
                                      "end": 3471
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3492,
                                        "end": 3502
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3492,
                                      "end": 3502
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 3523,
                                        "end": 3532
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3559,
                                              "end": 3561
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3559,
                                            "end": 3561
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 3586,
                                              "end": 3596
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3586,
                                            "end": 3596
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 3621,
                                              "end": 3631
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3621,
                                            "end": 3631
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3656,
                                              "end": 3660
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3656,
                                            "end": 3660
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3685,
                                              "end": 3696
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3685,
                                            "end": 3696
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 3721,
                                              "end": 3728
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3721,
                                            "end": 3728
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 3753,
                                              "end": 3758
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3753,
                                            "end": 3758
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 3783,
                                              "end": 3793
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3783,
                                            "end": 3793
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 3818,
                                              "end": 3831
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 3862,
                                                    "end": 3864
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3862,
                                                  "end": 3864
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 3893,
                                                    "end": 3903
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3893,
                                                  "end": 3903
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 3932,
                                                    "end": 3942
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3932,
                                                  "end": 3942
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 3971,
                                                    "end": 3975
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3971,
                                                  "end": 3975
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 4004,
                                                    "end": 4015
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4004,
                                                  "end": 4015
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 4044,
                                                    "end": 4051
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4044,
                                                  "end": 4051
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 4080,
                                                    "end": 4085
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4080,
                                                  "end": 4085
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 4114,
                                                    "end": 4124
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4114,
                                                  "end": 4124
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3832,
                                              "end": 4150
                                            }
                                          },
                                          "loc": {
                                            "start": 3818,
                                            "end": 4150
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3533,
                                        "end": 4172
                                      }
                                    },
                                    "loc": {
                                      "start": 3523,
                                      "end": 4172
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3416,
                                  "end": 4190
                                }
                              },
                              "loc": {
                                "start": 3403,
                                "end": 4190
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 4207,
                                  "end": 4219
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4242,
                                        "end": 4244
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4242,
                                      "end": 4244
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4265,
                                        "end": 4275
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4265,
                                      "end": 4275
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4296,
                                        "end": 4308
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4335,
                                              "end": 4337
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4335,
                                            "end": 4337
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4362,
                                              "end": 4370
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4362,
                                            "end": 4370
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4395,
                                              "end": 4406
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4395,
                                            "end": 4406
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4431,
                                              "end": 4435
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4431,
                                            "end": 4435
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4309,
                                        "end": 4457
                                      }
                                    },
                                    "loc": {
                                      "start": 4296,
                                      "end": 4457
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 4478,
                                        "end": 4487
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4514,
                                              "end": 4516
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4514,
                                            "end": 4516
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 4541,
                                              "end": 4546
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4541,
                                            "end": 4546
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 4571,
                                              "end": 4575
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4571,
                                            "end": 4575
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 4600,
                                              "end": 4607
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4600,
                                            "end": 4607
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 4632,
                                              "end": 4644
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 4675,
                                                    "end": 4677
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4675,
                                                  "end": 4677
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 4706,
                                                    "end": 4714
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4706,
                                                  "end": 4714
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 4743,
                                                    "end": 4754
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4743,
                                                  "end": 4754
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 4783,
                                                    "end": 4787
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4783,
                                                  "end": 4787
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4645,
                                              "end": 4813
                                            }
                                          },
                                          "loc": {
                                            "start": 4632,
                                            "end": 4813
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4488,
                                        "end": 4835
                                      }
                                    },
                                    "loc": {
                                      "start": 4478,
                                      "end": 4835
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4220,
                                  "end": 4853
                                }
                              },
                              "loc": {
                                "start": 4207,
                                "end": 4853
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 4870,
                                  "end": 4878
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "FragmentSpread",
                                    "name": {
                                      "kind": "Name",
                                      "value": "Schedule_common",
                                      "loc": {
                                        "start": 4904,
                                        "end": 4919
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 4901,
                                      "end": 4919
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4879,
                                  "end": 4937
                                }
                              },
                              "loc": {
                                "start": 4870,
                                "end": 4937
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4954,
                                  "end": 4956
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4954,
                                "end": 4956
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 4973,
                                  "end": 4977
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4973,
                                "end": 4977
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 4994,
                                  "end": 5005
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4994,
                                "end": 5005
                              }
                            }
                          ],
                          "loc": {
                            "start": 438,
                            "end": 5019
                          }
                        },
                        "loc": {
                          "start": 433,
                          "end": 5019
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 5032,
                            "end": 5045
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5032,
                          "end": 5045
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 5058,
                            "end": 5066
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5058,
                          "end": 5066
                        }
                      }
                    ],
                    "loc": {
                      "start": 419,
                      "end": 5076
                    }
                  },
                  "loc": {
                    "start": 403,
                    "end": 5076
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 5085,
                      "end": 5094
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5085,
                    "end": 5094
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 5103,
                      "end": 5116
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5131,
                            "end": 5133
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5131,
                          "end": 5133
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 5146,
                            "end": 5156
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5146,
                          "end": 5156
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 5169,
                            "end": 5179
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5169,
                          "end": 5179
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 5192,
                            "end": 5197
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5192,
                          "end": 5197
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 5210,
                            "end": 5224
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5210,
                          "end": 5224
                        }
                      }
                    ],
                    "loc": {
                      "start": 5117,
                      "end": 5234
                    }
                  },
                  "loc": {
                    "start": 5103,
                    "end": 5234
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 5243,
                      "end": 5253
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "filters",
                          "loc": {
                            "start": 5268,
                            "end": 5275
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 5294,
                                  "end": 5296
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5294,
                                "end": 5296
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 5313,
                                  "end": 5323
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5313,
                                "end": 5323
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 5340,
                                  "end": 5343
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 5366,
                                        "end": 5368
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5366,
                                      "end": 5368
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 5389,
                                        "end": 5399
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5389,
                                      "end": 5399
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 5420,
                                        "end": 5423
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5420,
                                      "end": 5423
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 5444,
                                        "end": 5453
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5444,
                                      "end": 5453
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5474,
                                        "end": 5486
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5513,
                                              "end": 5515
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5513,
                                            "end": 5515
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5540,
                                              "end": 5548
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5540,
                                            "end": 5548
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5573,
                                              "end": 5584
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5573,
                                            "end": 5584
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5487,
                                        "end": 5606
                                      }
                                    },
                                    "loc": {
                                      "start": 5474,
                                      "end": 5606
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 5627,
                                        "end": 5630
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isOwn",
                                            "loc": {
                                              "start": 5657,
                                              "end": 5662
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5657,
                                            "end": 5662
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 5687,
                                              "end": 5699
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5687,
                                            "end": 5699
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5631,
                                        "end": 5721
                                      }
                                    },
                                    "loc": {
                                      "start": 5627,
                                      "end": 5721
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5344,
                                  "end": 5739
                                }
                              },
                              "loc": {
                                "start": 5340,
                                "end": 5739
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 5756,
                                  "end": 5765
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "labels",
                                      "loc": {
                                        "start": 5788,
                                        "end": 5794
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5821,
                                              "end": 5823
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5821,
                                            "end": 5823
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 5848,
                                              "end": 5853
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5848,
                                            "end": 5853
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 5878,
                                              "end": 5883
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5878,
                                            "end": 5883
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5795,
                                        "end": 5905
                                      }
                                    },
                                    "loc": {
                                      "start": 5788,
                                      "end": 5905
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderList",
                                      "loc": {
                                        "start": 5926,
                                        "end": 5938
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5965,
                                              "end": 5967
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5965,
                                            "end": 5967
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 5992,
                                              "end": 6002
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5992,
                                            "end": 6002
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 6027,
                                              "end": 6037
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6027,
                                            "end": 6037
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminders",
                                            "loc": {
                                              "start": 6062,
                                              "end": 6071
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 6102,
                                                    "end": 6104
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6102,
                                                  "end": 6104
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 6133,
                                                    "end": 6143
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6133,
                                                  "end": 6143
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 6172,
                                                    "end": 6182
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6172,
                                                  "end": 6182
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 6211,
                                                    "end": 6215
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6211,
                                                  "end": 6215
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 6244,
                                                    "end": 6255
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6244,
                                                  "end": 6255
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 6284,
                                                    "end": 6291
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6284,
                                                  "end": 6291
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 6320,
                                                    "end": 6325
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6320,
                                                  "end": 6325
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 6354,
                                                    "end": 6364
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6354,
                                                  "end": 6364
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminderItems",
                                                  "loc": {
                                                    "start": 6393,
                                                    "end": 6406
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 6441,
                                                          "end": 6443
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6441,
                                                        "end": 6443
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 6476,
                                                          "end": 6486
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6476,
                                                        "end": 6486
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 6519,
                                                          "end": 6529
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6519,
                                                        "end": 6529
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 6562,
                                                          "end": 6566
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6562,
                                                        "end": 6566
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 6599,
                                                          "end": 6610
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6599,
                                                        "end": 6610
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 6643,
                                                          "end": 6650
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6643,
                                                        "end": 6650
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 6683,
                                                          "end": 6688
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6683,
                                                        "end": 6688
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 6721,
                                                          "end": 6731
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6721,
                                                        "end": 6731
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 6407,
                                                    "end": 6761
                                                  }
                                                },
                                                "loc": {
                                                  "start": 6393,
                                                  "end": 6761
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6072,
                                              "end": 6787
                                            }
                                          },
                                          "loc": {
                                            "start": 6062,
                                            "end": 6787
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5939,
                                        "end": 6809
                                      }
                                    },
                                    "loc": {
                                      "start": 5926,
                                      "end": 6809
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resourceList",
                                      "loc": {
                                        "start": 6830,
                                        "end": 6842
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 6869,
                                              "end": 6871
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6869,
                                            "end": 6871
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 6896,
                                              "end": 6906
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6896,
                                            "end": 6906
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 6931,
                                              "end": 6943
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 6974,
                                                    "end": 6976
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6974,
                                                  "end": 6976
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 7005,
                                                    "end": 7013
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7005,
                                                  "end": 7013
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 7042,
                                                    "end": 7053
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7042,
                                                  "end": 7053
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 7082,
                                                    "end": 7086
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7082,
                                                  "end": 7086
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6944,
                                              "end": 7112
                                            }
                                          },
                                          "loc": {
                                            "start": 6931,
                                            "end": 7112
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resources",
                                            "loc": {
                                              "start": 7137,
                                              "end": 7146
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 7177,
                                                    "end": 7179
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7177,
                                                  "end": 7179
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 7208,
                                                    "end": 7213
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7208,
                                                  "end": 7213
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "link",
                                                  "loc": {
                                                    "start": 7242,
                                                    "end": 7246
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7242,
                                                  "end": 7246
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "usedFor",
                                                  "loc": {
                                                    "start": 7275,
                                                    "end": 7282
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7275,
                                                  "end": 7282
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 7311,
                                                    "end": 7323
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 7358,
                                                          "end": 7360
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7358,
                                                        "end": 7360
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 7393,
                                                          "end": 7401
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7393,
                                                        "end": 7401
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 7434,
                                                          "end": 7445
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7434,
                                                        "end": 7445
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 7478,
                                                          "end": 7482
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7478,
                                                        "end": 7482
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 7324,
                                                    "end": 7512
                                                  }
                                                },
                                                "loc": {
                                                  "start": 7311,
                                                  "end": 7512
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 7147,
                                              "end": 7538
                                            }
                                          },
                                          "loc": {
                                            "start": 7137,
                                            "end": 7538
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6843,
                                        "end": 7560
                                      }
                                    },
                                    "loc": {
                                      "start": 6830,
                                      "end": 7560
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 7581,
                                        "end": 7589
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "FragmentSpread",
                                          "name": {
                                            "kind": "Name",
                                            "value": "Schedule_common",
                                            "loc": {
                                              "start": 7619,
                                              "end": 7634
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 7616,
                                            "end": 7634
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 7590,
                                        "end": 7656
                                      }
                                    },
                                    "loc": {
                                      "start": 7581,
                                      "end": 7656
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 7677,
                                        "end": 7679
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7677,
                                      "end": 7679
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 7700,
                                        "end": 7704
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7700,
                                      "end": 7704
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 7725,
                                        "end": 7736
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7725,
                                      "end": 7736
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5766,
                                  "end": 7754
                                }
                              },
                              "loc": {
                                "start": 5756,
                                "end": 7754
                              }
                            }
                          ],
                          "loc": {
                            "start": 5276,
                            "end": 7768
                          }
                        },
                        "loc": {
                          "start": 5268,
                          "end": 7768
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 7781,
                            "end": 7787
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 7806,
                                  "end": 7808
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7806,
                                "end": 7808
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 7825,
                                  "end": 7830
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7825,
                                "end": 7830
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 7847,
                                  "end": 7852
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7847,
                                "end": 7852
                              }
                            }
                          ],
                          "loc": {
                            "start": 7788,
                            "end": 7866
                          }
                        },
                        "loc": {
                          "start": 7781,
                          "end": 7866
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 7879,
                            "end": 7891
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 7910,
                                  "end": 7912
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7910,
                                "end": 7912
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 7929,
                                  "end": 7939
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7929,
                                "end": 7939
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 7956,
                                  "end": 7966
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7956,
                                "end": 7966
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 7983,
                                  "end": 7992
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 8015,
                                        "end": 8017
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8015,
                                      "end": 8017
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 8038,
                                        "end": 8048
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8038,
                                      "end": 8048
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 8069,
                                        "end": 8079
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8069,
                                      "end": 8079
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 8100,
                                        "end": 8104
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8100,
                                      "end": 8104
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 8125,
                                        "end": 8136
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8125,
                                      "end": 8136
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 8157,
                                        "end": 8164
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8157,
                                      "end": 8164
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 8185,
                                        "end": 8190
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8185,
                                      "end": 8190
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 8211,
                                        "end": 8221
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8211,
                                      "end": 8221
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 8242,
                                        "end": 8255
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 8282,
                                              "end": 8284
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8282,
                                            "end": 8284
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 8309,
                                              "end": 8319
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8309,
                                            "end": 8319
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 8344,
                                              "end": 8354
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8344,
                                            "end": 8354
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 8379,
                                              "end": 8383
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8379,
                                            "end": 8383
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 8408,
                                              "end": 8419
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8408,
                                            "end": 8419
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 8444,
                                              "end": 8451
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8444,
                                            "end": 8451
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 8476,
                                              "end": 8481
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8476,
                                            "end": 8481
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 8506,
                                              "end": 8516
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8506,
                                            "end": 8516
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 8256,
                                        "end": 8538
                                      }
                                    },
                                    "loc": {
                                      "start": 8242,
                                      "end": 8538
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 7993,
                                  "end": 8556
                                }
                              },
                              "loc": {
                                "start": 7983,
                                "end": 8556
                              }
                            }
                          ],
                          "loc": {
                            "start": 7892,
                            "end": 8570
                          }
                        },
                        "loc": {
                          "start": 7879,
                          "end": 8570
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 8583,
                            "end": 8595
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 8614,
                                  "end": 8616
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8614,
                                "end": 8616
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 8633,
                                  "end": 8643
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8633,
                                "end": 8643
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 8660,
                                  "end": 8672
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 8695,
                                        "end": 8697
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8695,
                                      "end": 8697
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 8718,
                                        "end": 8726
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8718,
                                      "end": 8726
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 8747,
                                        "end": 8758
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8747,
                                      "end": 8758
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 8779,
                                        "end": 8783
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8779,
                                      "end": 8783
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8673,
                                  "end": 8801
                                }
                              },
                              "loc": {
                                "start": 8660,
                                "end": 8801
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 8818,
                                  "end": 8827
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 8850,
                                        "end": 8852
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8850,
                                      "end": 8852
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 8873,
                                        "end": 8878
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8873,
                                      "end": 8878
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 8899,
                                        "end": 8903
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8899,
                                      "end": 8903
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 8924,
                                        "end": 8931
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8924,
                                      "end": 8931
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 8952,
                                        "end": 8964
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 8991,
                                              "end": 8993
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8991,
                                            "end": 8993
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 9018,
                                              "end": 9026
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9018,
                                            "end": 9026
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 9051,
                                              "end": 9062
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9051,
                                            "end": 9062
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 9087,
                                              "end": 9091
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9087,
                                            "end": 9091
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 8965,
                                        "end": 9113
                                      }
                                    },
                                    "loc": {
                                      "start": 8952,
                                      "end": 9113
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8828,
                                  "end": 9131
                                }
                              },
                              "loc": {
                                "start": 8818,
                                "end": 9131
                              }
                            }
                          ],
                          "loc": {
                            "start": 8596,
                            "end": 9145
                          }
                        },
                        "loc": {
                          "start": 8583,
                          "end": 9145
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 9158,
                            "end": 9166
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 9188,
                                  "end": 9203
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 9185,
                                "end": 9203
                              }
                            }
                          ],
                          "loc": {
                            "start": 9167,
                            "end": 9217
                          }
                        },
                        "loc": {
                          "start": 9158,
                          "end": 9217
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 9230,
                            "end": 9232
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9230,
                          "end": 9232
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 9245,
                            "end": 9249
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9245,
                          "end": 9249
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 9262,
                            "end": 9273
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9262,
                          "end": 9273
                        }
                      }
                    ],
                    "loc": {
                      "start": 5254,
                      "end": 9283
                    }
                  },
                  "loc": {
                    "start": 5243,
                    "end": 9283
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 9292,
                      "end": 9298
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9292,
                    "end": 9298
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 9307,
                      "end": 9317
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9307,
                    "end": 9317
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 9326,
                      "end": 9328
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9326,
                    "end": 9328
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 9337,
                      "end": 9346
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9337,
                    "end": 9346
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 9355,
                      "end": 9371
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9355,
                    "end": 9371
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 9380,
                      "end": 9384
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9380,
                    "end": 9384
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 9393,
                      "end": 9403
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9393,
                    "end": 9403
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 9412,
                      "end": 9425
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9412,
                    "end": 9425
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 9434,
                      "end": 9453
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9434,
                    "end": 9453
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 9462,
                      "end": 9475
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9462,
                    "end": 9475
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractsCount",
                    "loc": {
                      "start": 9484,
                      "end": 9503
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9484,
                    "end": 9503
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 9512,
                      "end": 9526
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9512,
                    "end": 9526
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 9535,
                      "end": 9540
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9535,
                    "end": 9540
                  }
                }
              ],
              "loc": {
                "start": 393,
                "end": 9546
              }
            },
            "loc": {
              "start": 387,
              "end": 9546
            }
          }
        ],
        "loc": {
          "start": 353,
          "end": 9550
        }
      },
      "loc": {
        "start": 327,
        "end": 9550
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 40,
          "end": 42
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 40,
        "end": 42
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 43,
          "end": 53
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 43,
        "end": 53
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 54,
          "end": 64
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 54,
        "end": 64
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 65,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 65,
        "end": 74
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 75,
          "end": 82
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 75,
        "end": 82
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 83,
          "end": 91
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 83,
        "end": 91
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 92,
          "end": 102
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 109,
                "end": 111
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 109,
              "end": 111
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 116,
                "end": 133
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 116,
              "end": 133
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 138,
                "end": 150
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 138,
              "end": 150
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 155,
                "end": 165
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 155,
              "end": 165
            }
          }
        ],
        "loc": {
          "start": 103,
          "end": 167
        }
      },
      "loc": {
        "start": 92,
        "end": 167
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 168,
          "end": 179
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 186,
                "end": 188
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 186,
              "end": 188
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 193,
                "end": 207
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 207
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 212,
                "end": 220
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 212,
              "end": 220
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 225,
                "end": 234
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 225,
              "end": 234
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 239,
                "end": 249
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 239,
              "end": 249
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 254,
                "end": 259
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 254,
              "end": 259
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 264,
                "end": 271
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 264,
              "end": 271
            }
          }
        ],
        "loc": {
          "start": 180,
          "end": 273
        }
      },
      "loc": {
        "start": 168,
        "end": 273
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 10,
          "end": 25
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 29,
            "end": 37
          }
        },
        "loc": {
          "start": 29,
          "end": 37
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 40,
                "end": 42
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 40,
              "end": 42
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 43,
                "end": 53
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 43,
              "end": 53
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 54,
                "end": 64
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 54,
              "end": 64
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 65,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 65,
              "end": 74
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 75,
                "end": 82
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 75,
              "end": 82
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 83,
                "end": 91
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 83,
              "end": 91
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 92,
                "end": 102
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 109,
                      "end": 111
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 109,
                    "end": 111
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 116,
                      "end": 133
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 116,
                    "end": 133
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 138,
                      "end": 150
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 138,
                    "end": 150
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 155,
                      "end": 165
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 155,
                    "end": 165
                  }
                }
              ],
              "loc": {
                "start": 103,
                "end": 167
              }
            },
            "loc": {
              "start": 92,
              "end": 167
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 168,
                "end": 179
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 186,
                      "end": 188
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 186,
                    "end": 188
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 193,
                      "end": 207
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 207
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 212,
                      "end": 220
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 212,
                    "end": 220
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 225,
                      "end": 234
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 225,
                    "end": 234
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 239,
                      "end": 249
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 239,
                    "end": 249
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 254,
                      "end": 259
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 254,
                    "end": 259
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 264,
                      "end": 271
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 264,
                    "end": 271
                  }
                }
              ],
              "loc": {
                "start": 180,
                "end": 273
              }
            },
            "loc": {
              "start": 168,
              "end": 273
            }
          }
        ],
        "loc": {
          "start": 38,
          "end": 275
        }
      },
      "loc": {
        "start": 1,
        "end": 275
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "emailLogIn",
      "loc": {
        "start": 286,
        "end": 296
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 298,
              "end": 303
            }
          },
          "loc": {
            "start": 297,
            "end": 303
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "EmailLogInInput",
              "loc": {
                "start": 305,
                "end": 320
              }
            },
            "loc": {
              "start": 305,
              "end": 320
            }
          },
          "loc": {
            "start": 305,
            "end": 321
          }
        },
        "directives": [],
        "loc": {
          "start": 297,
          "end": 321
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "emailLogIn",
            "loc": {
              "start": 327,
              "end": 337
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 338,
                  "end": 343
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 346,
                    "end": 351
                  }
                },
                "loc": {
                  "start": 345,
                  "end": 351
                }
              },
              "loc": {
                "start": 338,
                "end": 351
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "isLoggedIn",
                  "loc": {
                    "start": 359,
                    "end": 369
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 359,
                  "end": 369
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "timeZone",
                  "loc": {
                    "start": 374,
                    "end": 382
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 374,
                  "end": 382
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "users",
                  "loc": {
                    "start": 387,
                    "end": 392
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "activeFocusMode",
                        "loc": {
                          "start": 403,
                          "end": 418
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "mode",
                              "loc": {
                                "start": 433,
                                "end": 437
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filters",
                                    "loc": {
                                      "start": 456,
                                      "end": 463
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 486,
                                            "end": 488
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 486,
                                          "end": 488
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "filterType",
                                          "loc": {
                                            "start": 509,
                                            "end": 519
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 509,
                                          "end": 519
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 540,
                                            "end": 543
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 570,
                                                  "end": 572
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 570,
                                                "end": 572
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 597,
                                                  "end": 607
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 597,
                                                "end": 607
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "tag",
                                                "loc": {
                                                  "start": 632,
                                                  "end": 635
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 632,
                                                "end": 635
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "bookmarks",
                                                "loc": {
                                                  "start": 660,
                                                  "end": 669
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 660,
                                                "end": 669
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 694,
                                                  "end": 706
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 737,
                                                        "end": 739
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 737,
                                                      "end": 739
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 768,
                                                        "end": 776
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 768,
                                                      "end": 776
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 805,
                                                        "end": 816
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 805,
                                                      "end": 816
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 707,
                                                  "end": 842
                                                }
                                              },
                                              "loc": {
                                                "start": 694,
                                                "end": 842
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "you",
                                                "loc": {
                                                  "start": 867,
                                                  "end": 870
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isOwn",
                                                      "loc": {
                                                        "start": 901,
                                                        "end": 906
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 901,
                                                      "end": 906
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isBookmarked",
                                                      "loc": {
                                                        "start": 935,
                                                        "end": 947
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 935,
                                                      "end": 947
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 871,
                                                  "end": 973
                                                }
                                              },
                                              "loc": {
                                                "start": 867,
                                                "end": 973
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 544,
                                            "end": 995
                                          }
                                        },
                                        "loc": {
                                          "start": 540,
                                          "end": 995
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "focusMode",
                                          "loc": {
                                            "start": 1016,
                                            "end": 1025
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "labels",
                                                "loc": {
                                                  "start": 1052,
                                                  "end": 1058
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 1089,
                                                        "end": 1091
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1089,
                                                      "end": 1091
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "color",
                                                      "loc": {
                                                        "start": 1120,
                                                        "end": 1125
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1120,
                                                      "end": 1125
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "label",
                                                      "loc": {
                                                        "start": 1154,
                                                        "end": 1159
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1154,
                                                      "end": 1159
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1059,
                                                  "end": 1185
                                                }
                                              },
                                              "loc": {
                                                "start": 1052,
                                                "end": 1185
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderList",
                                                "loc": {
                                                  "start": 1210,
                                                  "end": 1222
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 1253,
                                                        "end": 1255
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1253,
                                                      "end": 1255
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 1284,
                                                        "end": 1294
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1284,
                                                      "end": 1294
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 1323,
                                                        "end": 1333
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1323,
                                                      "end": 1333
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "reminders",
                                                      "loc": {
                                                        "start": 1362,
                                                        "end": 1371
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "selectionSet": {
                                                      "kind": "SelectionSet",
                                                      "selections": [
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "id",
                                                            "loc": {
                                                              "start": 1406,
                                                              "end": 1408
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1406,
                                                            "end": 1408
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "created_at",
                                                            "loc": {
                                                              "start": 1441,
                                                              "end": 1451
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1441,
                                                            "end": 1451
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "updated_at",
                                                            "loc": {
                                                              "start": 1484,
                                                              "end": 1494
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1484,
                                                            "end": 1494
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 1527,
                                                              "end": 1531
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1527,
                                                            "end": 1531
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 1564,
                                                              "end": 1575
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1564,
                                                            "end": 1575
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "dueDate",
                                                            "loc": {
                                                              "start": 1608,
                                                              "end": 1615
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1608,
                                                            "end": 1615
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 1648,
                                                              "end": 1653
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1648,
                                                            "end": 1653
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "isComplete",
                                                            "loc": {
                                                              "start": 1686,
                                                              "end": 1696
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1686,
                                                            "end": 1696
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "reminderItems",
                                                            "loc": {
                                                              "start": 1729,
                                                              "end": 1742
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "selectionSet": {
                                                            "kind": "SelectionSet",
                                                            "selections": [
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "id",
                                                                  "loc": {
                                                                    "start": 1781,
                                                                    "end": 1783
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1781,
                                                                  "end": 1783
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "created_at",
                                                                  "loc": {
                                                                    "start": 1820,
                                                                    "end": 1830
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1820,
                                                                  "end": 1830
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "updated_at",
                                                                  "loc": {
                                                                    "start": 1867,
                                                                    "end": 1877
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1867,
                                                                  "end": 1877
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "name",
                                                                  "loc": {
                                                                    "start": 1914,
                                                                    "end": 1918
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1914,
                                                                  "end": 1918
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "description",
                                                                  "loc": {
                                                                    "start": 1955,
                                                                    "end": 1966
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1955,
                                                                  "end": 1966
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "dueDate",
                                                                  "loc": {
                                                                    "start": 2003,
                                                                    "end": 2010
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2003,
                                                                  "end": 2010
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "index",
                                                                  "loc": {
                                                                    "start": 2047,
                                                                    "end": 2052
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2047,
                                                                  "end": 2052
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "isComplete",
                                                                  "loc": {
                                                                    "start": 2089,
                                                                    "end": 2099
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2089,
                                                                  "end": 2099
                                                                }
                                                              }
                                                            ],
                                                            "loc": {
                                                              "start": 1743,
                                                              "end": 2133
                                                            }
                                                          },
                                                          "loc": {
                                                            "start": 1729,
                                                            "end": 2133
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 1372,
                                                        "end": 2163
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 1362,
                                                      "end": 2163
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1223,
                                                  "end": 2189
                                                }
                                              },
                                              "loc": {
                                                "start": 1210,
                                                "end": 2189
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "resourceList",
                                                "loc": {
                                                  "start": 2214,
                                                  "end": 2226
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 2257,
                                                        "end": 2259
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2257,
                                                      "end": 2259
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 2288,
                                                        "end": 2298
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2288,
                                                      "end": 2298
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "translations",
                                                      "loc": {
                                                        "start": 2327,
                                                        "end": 2339
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "selectionSet": {
                                                      "kind": "SelectionSet",
                                                      "selections": [
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "id",
                                                            "loc": {
                                                              "start": 2374,
                                                              "end": 2376
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2374,
                                                            "end": 2376
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "language",
                                                            "loc": {
                                                              "start": 2409,
                                                              "end": 2417
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2409,
                                                            "end": 2417
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 2450,
                                                              "end": 2461
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2450,
                                                            "end": 2461
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 2494,
                                                              "end": 2498
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2494,
                                                            "end": 2498
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 2340,
                                                        "end": 2528
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 2327,
                                                      "end": 2528
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "resources",
                                                      "loc": {
                                                        "start": 2557,
                                                        "end": 2566
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "selectionSet": {
                                                      "kind": "SelectionSet",
                                                      "selections": [
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "id",
                                                            "loc": {
                                                              "start": 2601,
                                                              "end": 2603
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2601,
                                                            "end": 2603
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 2636,
                                                              "end": 2641
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2636,
                                                            "end": 2641
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "link",
                                                            "loc": {
                                                              "start": 2674,
                                                              "end": 2678
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2674,
                                                            "end": 2678
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "usedFor",
                                                            "loc": {
                                                              "start": 2711,
                                                              "end": 2718
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2711,
                                                            "end": 2718
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "translations",
                                                            "loc": {
                                                              "start": 2751,
                                                              "end": 2763
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "selectionSet": {
                                                            "kind": "SelectionSet",
                                                            "selections": [
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "id",
                                                                  "loc": {
                                                                    "start": 2802,
                                                                    "end": 2804
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2802,
                                                                  "end": 2804
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "language",
                                                                  "loc": {
                                                                    "start": 2841,
                                                                    "end": 2849
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2841,
                                                                  "end": 2849
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "description",
                                                                  "loc": {
                                                                    "start": 2886,
                                                                    "end": 2897
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2886,
                                                                  "end": 2897
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "name",
                                                                  "loc": {
                                                                    "start": 2934,
                                                                    "end": 2938
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2934,
                                                                  "end": 2938
                                                                }
                                                              }
                                                            ],
                                                            "loc": {
                                                              "start": 2764,
                                                              "end": 2972
                                                            }
                                                          },
                                                          "loc": {
                                                            "start": 2751,
                                                            "end": 2972
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 2567,
                                                        "end": 3002
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 2557,
                                                      "end": 3002
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 2227,
                                                  "end": 3028
                                                }
                                              },
                                              "loc": {
                                                "start": 2214,
                                                "end": 3028
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "schedule",
                                                "loc": {
                                                  "start": 3053,
                                                  "end": 3061
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "FragmentSpread",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "Schedule_common",
                                                      "loc": {
                                                        "start": 3095,
                                                        "end": 3110
                                                      }
                                                    },
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3092,
                                                      "end": 3110
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3062,
                                                  "end": 3136
                                                }
                                              },
                                              "loc": {
                                                "start": 3053,
                                                "end": 3136
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 3161,
                                                  "end": 3163
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3161,
                                                "end": 3163
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 3188,
                                                  "end": 3192
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3188,
                                                "end": 3192
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3217,
                                                  "end": 3228
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3217,
                                                "end": 3228
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1026,
                                            "end": 3250
                                          }
                                        },
                                        "loc": {
                                          "start": 1016,
                                          "end": 3250
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 464,
                                      "end": 3268
                                    }
                                  },
                                  "loc": {
                                    "start": 456,
                                    "end": 3268
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "labels",
                                    "loc": {
                                      "start": 3285,
                                      "end": 3291
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 3314,
                                            "end": 3316
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3314,
                                          "end": 3316
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "color",
                                          "loc": {
                                            "start": 3337,
                                            "end": 3342
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3337,
                                          "end": 3342
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "label",
                                          "loc": {
                                            "start": 3363,
                                            "end": 3368
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3363,
                                          "end": 3368
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3292,
                                      "end": 3386
                                    }
                                  },
                                  "loc": {
                                    "start": 3285,
                                    "end": 3386
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminderList",
                                    "loc": {
                                      "start": 3403,
                                      "end": 3415
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 3438,
                                            "end": 3440
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3438,
                                          "end": 3440
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 3461,
                                            "end": 3471
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3461,
                                          "end": 3471
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 3492,
                                            "end": 3502
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3492,
                                          "end": 3502
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminders",
                                          "loc": {
                                            "start": 3523,
                                            "end": 3532
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 3559,
                                                  "end": 3561
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3559,
                                                "end": 3561
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 3586,
                                                  "end": 3596
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3586,
                                                "end": 3596
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 3621,
                                                  "end": 3631
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3621,
                                                "end": 3631
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 3656,
                                                  "end": 3660
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3656,
                                                "end": 3660
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3685,
                                                  "end": 3696
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3685,
                                                "end": 3696
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 3721,
                                                  "end": 3728
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3721,
                                                "end": 3728
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 3753,
                                                  "end": 3758
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3753,
                                                "end": 3758
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 3783,
                                                  "end": 3793
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3783,
                                                "end": 3793
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderItems",
                                                "loc": {
                                                  "start": 3818,
                                                  "end": 3831
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 3862,
                                                        "end": 3864
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3862,
                                                      "end": 3864
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 3893,
                                                        "end": 3903
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3893,
                                                      "end": 3903
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 3932,
                                                        "end": 3942
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3932,
                                                      "end": 3942
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 3971,
                                                        "end": 3975
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3971,
                                                      "end": 3975
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 4004,
                                                        "end": 4015
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4004,
                                                      "end": 4015
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 4044,
                                                        "end": 4051
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4044,
                                                      "end": 4051
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 4080,
                                                        "end": 4085
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4080,
                                                      "end": 4085
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 4114,
                                                        "end": 4124
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4114,
                                                      "end": 4124
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3832,
                                                  "end": 4150
                                                }
                                              },
                                              "loc": {
                                                "start": 3818,
                                                "end": 4150
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3533,
                                            "end": 4172
                                          }
                                        },
                                        "loc": {
                                          "start": 3523,
                                          "end": 4172
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3416,
                                      "end": 4190
                                    }
                                  },
                                  "loc": {
                                    "start": 3403,
                                    "end": 4190
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resourceList",
                                    "loc": {
                                      "start": 4207,
                                      "end": 4219
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 4242,
                                            "end": 4244
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4242,
                                          "end": 4244
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 4265,
                                            "end": 4275
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4265,
                                          "end": 4275
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 4296,
                                            "end": 4308
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 4335,
                                                  "end": 4337
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4335,
                                                "end": 4337
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 4362,
                                                  "end": 4370
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4362,
                                                "end": 4370
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 4395,
                                                  "end": 4406
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4395,
                                                "end": 4406
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 4431,
                                                  "end": 4435
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4431,
                                                "end": 4435
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4309,
                                            "end": 4457
                                          }
                                        },
                                        "loc": {
                                          "start": 4296,
                                          "end": 4457
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resources",
                                          "loc": {
                                            "start": 4478,
                                            "end": 4487
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 4514,
                                                  "end": 4516
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4514,
                                                "end": 4516
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 4541,
                                                  "end": 4546
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4541,
                                                "end": 4546
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "link",
                                                "loc": {
                                                  "start": 4571,
                                                  "end": 4575
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4571,
                                                "end": 4575
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "usedFor",
                                                "loc": {
                                                  "start": 4600,
                                                  "end": 4607
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4600,
                                                "end": 4607
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 4632,
                                                  "end": 4644
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 4675,
                                                        "end": 4677
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4675,
                                                      "end": 4677
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 4706,
                                                        "end": 4714
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4706,
                                                      "end": 4714
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 4743,
                                                        "end": 4754
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4743,
                                                      "end": 4754
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 4783,
                                                        "end": 4787
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4783,
                                                      "end": 4787
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 4645,
                                                  "end": 4813
                                                }
                                              },
                                              "loc": {
                                                "start": 4632,
                                                "end": 4813
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4488,
                                            "end": 4835
                                          }
                                        },
                                        "loc": {
                                          "start": 4478,
                                          "end": 4835
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4220,
                                      "end": 4853
                                    }
                                  },
                                  "loc": {
                                    "start": 4207,
                                    "end": 4853
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "schedule",
                                    "loc": {
                                      "start": 4870,
                                      "end": 4878
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "FragmentSpread",
                                        "name": {
                                          "kind": "Name",
                                          "value": "Schedule_common",
                                          "loc": {
                                            "start": 4904,
                                            "end": 4919
                                          }
                                        },
                                        "directives": [],
                                        "loc": {
                                          "start": 4901,
                                          "end": 4919
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4879,
                                      "end": 4937
                                    }
                                  },
                                  "loc": {
                                    "start": 4870,
                                    "end": 4937
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 4954,
                                      "end": 4956
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4954,
                                    "end": 4956
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "name",
                                    "loc": {
                                      "start": 4973,
                                      "end": 4977
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4973,
                                    "end": 4977
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "description",
                                    "loc": {
                                      "start": 4994,
                                      "end": 5005
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4994,
                                    "end": 5005
                                  }
                                }
                              ],
                              "loc": {
                                "start": 438,
                                "end": 5019
                              }
                            },
                            "loc": {
                              "start": 433,
                              "end": 5019
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopCondition",
                              "loc": {
                                "start": 5032,
                                "end": 5045
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5032,
                              "end": 5045
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopTime",
                              "loc": {
                                "start": 5058,
                                "end": 5066
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5058,
                              "end": 5066
                            }
                          }
                        ],
                        "loc": {
                          "start": 419,
                          "end": 5076
                        }
                      },
                      "loc": {
                        "start": 403,
                        "end": 5076
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "apisCount",
                        "loc": {
                          "start": 5085,
                          "end": 5094
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5085,
                        "end": 5094
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "bookmarkLists",
                        "loc": {
                          "start": 5103,
                          "end": 5116
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 5131,
                                "end": 5133
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5131,
                              "end": 5133
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "created_at",
                              "loc": {
                                "start": 5146,
                                "end": 5156
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5146,
                              "end": 5156
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "updated_at",
                              "loc": {
                                "start": 5169,
                                "end": 5179
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5169,
                              "end": 5179
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "label",
                              "loc": {
                                "start": 5192,
                                "end": 5197
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5192,
                              "end": 5197
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "bookmarksCount",
                              "loc": {
                                "start": 5210,
                                "end": 5224
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5210,
                              "end": 5224
                            }
                          }
                        ],
                        "loc": {
                          "start": 5117,
                          "end": 5234
                        }
                      },
                      "loc": {
                        "start": 5103,
                        "end": 5234
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "focusModes",
                        "loc": {
                          "start": 5243,
                          "end": 5253
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "filters",
                              "loc": {
                                "start": 5268,
                                "end": 5275
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 5294,
                                      "end": 5296
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5294,
                                    "end": 5296
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filterType",
                                    "loc": {
                                      "start": 5313,
                                      "end": 5323
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5313,
                                    "end": 5323
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "tag",
                                    "loc": {
                                      "start": 5340,
                                      "end": 5343
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 5366,
                                            "end": 5368
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5366,
                                          "end": 5368
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 5389,
                                            "end": 5399
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5389,
                                          "end": 5399
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 5420,
                                            "end": 5423
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5420,
                                          "end": 5423
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "bookmarks",
                                          "loc": {
                                            "start": 5444,
                                            "end": 5453
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5444,
                                          "end": 5453
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 5474,
                                            "end": 5486
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 5513,
                                                  "end": 5515
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5513,
                                                "end": 5515
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 5540,
                                                  "end": 5548
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5540,
                                                "end": 5548
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 5573,
                                                  "end": 5584
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5573,
                                                "end": 5584
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5487,
                                            "end": 5606
                                          }
                                        },
                                        "loc": {
                                          "start": 5474,
                                          "end": 5606
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "you",
                                          "loc": {
                                            "start": 5627,
                                            "end": 5630
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isOwn",
                                                "loc": {
                                                  "start": 5657,
                                                  "end": 5662
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5657,
                                                "end": 5662
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isBookmarked",
                                                "loc": {
                                                  "start": 5687,
                                                  "end": 5699
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5687,
                                                "end": 5699
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5631,
                                            "end": 5721
                                          }
                                        },
                                        "loc": {
                                          "start": 5627,
                                          "end": 5721
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5344,
                                      "end": 5739
                                    }
                                  },
                                  "loc": {
                                    "start": 5340,
                                    "end": 5739
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "focusMode",
                                    "loc": {
                                      "start": 5756,
                                      "end": 5765
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "labels",
                                          "loc": {
                                            "start": 5788,
                                            "end": 5794
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 5821,
                                                  "end": 5823
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5821,
                                                "end": 5823
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "color",
                                                "loc": {
                                                  "start": 5848,
                                                  "end": 5853
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5848,
                                                "end": 5853
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "label",
                                                "loc": {
                                                  "start": 5878,
                                                  "end": 5883
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5878,
                                                "end": 5883
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5795,
                                            "end": 5905
                                          }
                                        },
                                        "loc": {
                                          "start": 5788,
                                          "end": 5905
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderList",
                                          "loc": {
                                            "start": 5926,
                                            "end": 5938
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 5965,
                                                  "end": 5967
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5965,
                                                "end": 5967
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 5992,
                                                  "end": 6002
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5992,
                                                "end": 6002
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 6027,
                                                  "end": 6037
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6027,
                                                "end": 6037
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminders",
                                                "loc": {
                                                  "start": 6062,
                                                  "end": 6071
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 6102,
                                                        "end": 6104
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6102,
                                                      "end": 6104
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 6133,
                                                        "end": 6143
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6133,
                                                      "end": 6143
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 6172,
                                                        "end": 6182
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6172,
                                                      "end": 6182
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 6211,
                                                        "end": 6215
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6211,
                                                      "end": 6215
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 6244,
                                                        "end": 6255
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6244,
                                                      "end": 6255
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 6284,
                                                        "end": 6291
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6284,
                                                      "end": 6291
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 6320,
                                                        "end": 6325
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6320,
                                                      "end": 6325
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 6354,
                                                        "end": 6364
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6354,
                                                      "end": 6364
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "reminderItems",
                                                      "loc": {
                                                        "start": 6393,
                                                        "end": 6406
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "selectionSet": {
                                                      "kind": "SelectionSet",
                                                      "selections": [
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "id",
                                                            "loc": {
                                                              "start": 6441,
                                                              "end": 6443
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6441,
                                                            "end": 6443
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "created_at",
                                                            "loc": {
                                                              "start": 6476,
                                                              "end": 6486
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6476,
                                                            "end": 6486
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "updated_at",
                                                            "loc": {
                                                              "start": 6519,
                                                              "end": 6529
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6519,
                                                            "end": 6529
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 6562,
                                                              "end": 6566
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6562,
                                                            "end": 6566
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 6599,
                                                              "end": 6610
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6599,
                                                            "end": 6610
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "dueDate",
                                                            "loc": {
                                                              "start": 6643,
                                                              "end": 6650
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6643,
                                                            "end": 6650
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 6683,
                                                              "end": 6688
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6683,
                                                            "end": 6688
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "isComplete",
                                                            "loc": {
                                                              "start": 6721,
                                                              "end": 6731
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6721,
                                                            "end": 6731
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 6407,
                                                        "end": 6761
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 6393,
                                                      "end": 6761
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 6072,
                                                  "end": 6787
                                                }
                                              },
                                              "loc": {
                                                "start": 6062,
                                                "end": 6787
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5939,
                                            "end": 6809
                                          }
                                        },
                                        "loc": {
                                          "start": 5926,
                                          "end": 6809
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resourceList",
                                          "loc": {
                                            "start": 6830,
                                            "end": 6842
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 6869,
                                                  "end": 6871
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6869,
                                                "end": 6871
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 6896,
                                                  "end": 6906
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6896,
                                                "end": 6906
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 6931,
                                                  "end": 6943
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 6974,
                                                        "end": 6976
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6974,
                                                      "end": 6976
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 7005,
                                                        "end": 7013
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7005,
                                                      "end": 7013
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 7042,
                                                        "end": 7053
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7042,
                                                      "end": 7053
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 7082,
                                                        "end": 7086
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7082,
                                                      "end": 7086
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 6944,
                                                  "end": 7112
                                                }
                                              },
                                              "loc": {
                                                "start": 6931,
                                                "end": 7112
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "resources",
                                                "loc": {
                                                  "start": 7137,
                                                  "end": 7146
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 7177,
                                                        "end": 7179
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7177,
                                                      "end": 7179
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 7208,
                                                        "end": 7213
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7208,
                                                      "end": 7213
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "link",
                                                      "loc": {
                                                        "start": 7242,
                                                        "end": 7246
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7242,
                                                      "end": 7246
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "usedFor",
                                                      "loc": {
                                                        "start": 7275,
                                                        "end": 7282
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7275,
                                                      "end": 7282
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "translations",
                                                      "loc": {
                                                        "start": 7311,
                                                        "end": 7323
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "selectionSet": {
                                                      "kind": "SelectionSet",
                                                      "selections": [
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "id",
                                                            "loc": {
                                                              "start": 7358,
                                                              "end": 7360
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7358,
                                                            "end": 7360
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "language",
                                                            "loc": {
                                                              "start": 7393,
                                                              "end": 7401
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7393,
                                                            "end": 7401
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 7434,
                                                              "end": 7445
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7434,
                                                            "end": 7445
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 7478,
                                                              "end": 7482
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7478,
                                                            "end": 7482
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 7324,
                                                        "end": 7512
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 7311,
                                                      "end": 7512
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 7147,
                                                  "end": 7538
                                                }
                                              },
                                              "loc": {
                                                "start": 7137,
                                                "end": 7538
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 6843,
                                            "end": 7560
                                          }
                                        },
                                        "loc": {
                                          "start": 6830,
                                          "end": 7560
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "schedule",
                                          "loc": {
                                            "start": 7581,
                                            "end": 7589
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "FragmentSpread",
                                              "name": {
                                                "kind": "Name",
                                                "value": "Schedule_common",
                                                "loc": {
                                                  "start": 7619,
                                                  "end": 7634
                                                }
                                              },
                                              "directives": [],
                                              "loc": {
                                                "start": 7616,
                                                "end": 7634
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 7590,
                                            "end": 7656
                                          }
                                        },
                                        "loc": {
                                          "start": 7581,
                                          "end": 7656
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 7677,
                                            "end": 7679
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7677,
                                          "end": 7679
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 7700,
                                            "end": 7704
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7700,
                                          "end": 7704
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 7725,
                                            "end": 7736
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7725,
                                          "end": 7736
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5766,
                                      "end": 7754
                                    }
                                  },
                                  "loc": {
                                    "start": 5756,
                                    "end": 7754
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5276,
                                "end": 7768
                              }
                            },
                            "loc": {
                              "start": 5268,
                              "end": 7768
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "labels",
                              "loc": {
                                "start": 7781,
                                "end": 7787
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 7806,
                                      "end": 7808
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7806,
                                    "end": 7808
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "color",
                                    "loc": {
                                      "start": 7825,
                                      "end": 7830
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7825,
                                    "end": 7830
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "label",
                                    "loc": {
                                      "start": 7847,
                                      "end": 7852
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7847,
                                    "end": 7852
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7788,
                                "end": 7866
                              }
                            },
                            "loc": {
                              "start": 7781,
                              "end": 7866
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "reminderList",
                              "loc": {
                                "start": 7879,
                                "end": 7891
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 7910,
                                      "end": 7912
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7910,
                                    "end": 7912
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 7929,
                                      "end": 7939
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7929,
                                    "end": 7939
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "updated_at",
                                    "loc": {
                                      "start": 7956,
                                      "end": 7966
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7956,
                                    "end": 7966
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminders",
                                    "loc": {
                                      "start": 7983,
                                      "end": 7992
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 8015,
                                            "end": 8017
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8015,
                                          "end": 8017
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 8038,
                                            "end": 8048
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8038,
                                          "end": 8048
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 8069,
                                            "end": 8079
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8069,
                                          "end": 8079
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 8100,
                                            "end": 8104
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8100,
                                          "end": 8104
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 8125,
                                            "end": 8136
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8125,
                                          "end": 8136
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "dueDate",
                                          "loc": {
                                            "start": 8157,
                                            "end": 8164
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8157,
                                          "end": 8164
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 8185,
                                            "end": 8190
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8185,
                                          "end": 8190
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isComplete",
                                          "loc": {
                                            "start": 8211,
                                            "end": 8221
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8211,
                                          "end": 8221
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderItems",
                                          "loc": {
                                            "start": 8242,
                                            "end": 8255
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 8282,
                                                  "end": 8284
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8282,
                                                "end": 8284
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 8309,
                                                  "end": 8319
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8309,
                                                "end": 8319
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 8344,
                                                  "end": 8354
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8344,
                                                "end": 8354
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 8379,
                                                  "end": 8383
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8379,
                                                "end": 8383
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 8408,
                                                  "end": 8419
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8408,
                                                "end": 8419
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 8444,
                                                  "end": 8451
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8444,
                                                "end": 8451
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 8476,
                                                  "end": 8481
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8476,
                                                "end": 8481
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 8506,
                                                  "end": 8516
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8506,
                                                "end": 8516
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 8256,
                                            "end": 8538
                                          }
                                        },
                                        "loc": {
                                          "start": 8242,
                                          "end": 8538
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 7993,
                                      "end": 8556
                                    }
                                  },
                                  "loc": {
                                    "start": 7983,
                                    "end": 8556
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7892,
                                "end": 8570
                              }
                            },
                            "loc": {
                              "start": 7879,
                              "end": 8570
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "resourceList",
                              "loc": {
                                "start": 8583,
                                "end": 8595
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 8614,
                                      "end": 8616
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 8614,
                                    "end": 8616
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 8633,
                                      "end": 8643
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 8633,
                                    "end": 8643
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "translations",
                                    "loc": {
                                      "start": 8660,
                                      "end": 8672
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 8695,
                                            "end": 8697
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8695,
                                          "end": 8697
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "language",
                                          "loc": {
                                            "start": 8718,
                                            "end": 8726
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8718,
                                          "end": 8726
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 8747,
                                            "end": 8758
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8747,
                                          "end": 8758
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 8779,
                                            "end": 8783
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8779,
                                          "end": 8783
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 8673,
                                      "end": 8801
                                    }
                                  },
                                  "loc": {
                                    "start": 8660,
                                    "end": 8801
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resources",
                                    "loc": {
                                      "start": 8818,
                                      "end": 8827
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 8850,
                                            "end": 8852
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8850,
                                          "end": 8852
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 8873,
                                            "end": 8878
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8873,
                                          "end": 8878
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "link",
                                          "loc": {
                                            "start": 8899,
                                            "end": 8903
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8899,
                                          "end": 8903
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "usedFor",
                                          "loc": {
                                            "start": 8924,
                                            "end": 8931
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8924,
                                          "end": 8931
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 8952,
                                            "end": 8964
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 8991,
                                                  "end": 8993
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8991,
                                                "end": 8993
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 9018,
                                                  "end": 9026
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9018,
                                                "end": 9026
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 9051,
                                                  "end": 9062
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9051,
                                                "end": 9062
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 9087,
                                                  "end": 9091
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9087,
                                                "end": 9091
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 8965,
                                            "end": 9113
                                          }
                                        },
                                        "loc": {
                                          "start": 8952,
                                          "end": 9113
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 8828,
                                      "end": 9131
                                    }
                                  },
                                  "loc": {
                                    "start": 8818,
                                    "end": 9131
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8596,
                                "end": 9145
                              }
                            },
                            "loc": {
                              "start": 8583,
                              "end": 9145
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "schedule",
                              "loc": {
                                "start": 9158,
                                "end": 9166
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Schedule_common",
                                    "loc": {
                                      "start": 9188,
                                      "end": 9203
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 9185,
                                    "end": 9203
                                  }
                                }
                              ],
                              "loc": {
                                "start": 9167,
                                "end": 9217
                              }
                            },
                            "loc": {
                              "start": 9158,
                              "end": 9217
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 9230,
                                "end": 9232
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9230,
                              "end": 9232
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "name",
                              "loc": {
                                "start": 9245,
                                "end": 9249
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9245,
                              "end": 9249
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "description",
                              "loc": {
                                "start": 9262,
                                "end": 9273
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9262,
                              "end": 9273
                            }
                          }
                        ],
                        "loc": {
                          "start": 5254,
                          "end": 9283
                        }
                      },
                      "loc": {
                        "start": 5243,
                        "end": 9283
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "handle",
                        "loc": {
                          "start": 9292,
                          "end": 9298
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9292,
                        "end": 9298
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasPremium",
                        "loc": {
                          "start": 9307,
                          "end": 9317
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9307,
                        "end": 9317
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "id",
                        "loc": {
                          "start": 9326,
                          "end": 9328
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9326,
                        "end": 9328
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "languages",
                        "loc": {
                          "start": 9337,
                          "end": 9346
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9337,
                        "end": 9346
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "membershipsCount",
                        "loc": {
                          "start": 9355,
                          "end": 9371
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9355,
                        "end": 9371
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "name",
                        "loc": {
                          "start": 9380,
                          "end": 9384
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9380,
                        "end": 9384
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "notesCount",
                        "loc": {
                          "start": 9393,
                          "end": 9403
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9393,
                        "end": 9403
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "projectsCount",
                        "loc": {
                          "start": 9412,
                          "end": 9425
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9412,
                        "end": 9425
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "questionsAskedCount",
                        "loc": {
                          "start": 9434,
                          "end": 9453
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9434,
                        "end": 9453
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "routinesCount",
                        "loc": {
                          "start": 9462,
                          "end": 9475
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9462,
                        "end": 9475
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "smartContractsCount",
                        "loc": {
                          "start": 9484,
                          "end": 9503
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9484,
                        "end": 9503
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "standardsCount",
                        "loc": {
                          "start": 9512,
                          "end": 9526
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9512,
                        "end": 9526
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "theme",
                        "loc": {
                          "start": 9535,
                          "end": 9540
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9535,
                        "end": 9540
                      }
                    }
                  ],
                  "loc": {
                    "start": 393,
                    "end": 9546
                  }
                },
                "loc": {
                  "start": 387,
                  "end": 9546
                }
              }
            ],
            "loc": {
              "start": 353,
              "end": 9550
            }
          },
          "loc": {
            "start": 327,
            "end": 9550
          }
        }
      ],
      "loc": {
        "start": 323,
        "end": 9552
      }
    },
    "loc": {
      "start": 277,
      "end": 9552
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_emailLogIn"
  }
} as const;
